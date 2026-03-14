<?xml version="1.0" encoding="utf-8"?>
<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns="http://www.w3.org/1999/xhtml" xmlns:h="http://www.w3.org/1999/xhtml"
  xmlns:f="http://docbook.org/xslt/extension/namespace" exclude-result-prefixes="h f" version="2.0">

  <!-- This stylesheet crashes out at run-time saying that () isn't allowed as the 
       second argument of subsequence() -->
  <?spec xslt#stylesheet-functions?>
  <?error XPTY0004?>

  <xsl:output method="xml" encoding="utf-8" indent="yes"/>

  <xsl:variable name="nested">
    <p>
      <a href="outer" xml:id="foo"> outer <a href="inner"> inner <a href="inner inner"> inner inner
          </a>
        </a> outer <a href="inner2"> inner </a>
      </a>
    </p>
  </xsl:variable>

  <xsl:template match="/">
    <xsl:apply-templates select="$nested/h:p"/>
  </xsl:template>

  <xsl:template match="h:p">
    <p>
      <xsl:apply-templates/>
    </p>
  </xsl:template>

  <xsl:template match="h:a">
    <xsl:variable name="a">
      <a>
        <xsl:copy-of select="@*"/>
        <xsl:apply-templates/>
      </a>
    </xsl:variable>

    <xsl:choose>
      <xsl:when test="$a/h:a//h:a">
        <xsl:copy-of select="f:unwrap-anchors($a, 1)"/>
      </xsl:when>
      <xsl:otherwise>
        <xsl:copy-of select="$a"/>
      </xsl:otherwise>
    </xsl:choose>
  </xsl:template>

  <xsl:template match="comment() | processing-instruction() | text()">
    <xsl:copy/>
  </xsl:template>

  <xsl:function name="f:unwrap-anchors">
    <xsl:param name="contentseq"/>
    <xsl:param name="top-level"/>

    <xsl:message use-when="false()">
      <xsl:text>unwrap: </xsl:text>
      <xsl:value-of select="count($contentseq)"/>
      <xsl:text>, </xsl:text>
      <xsl:value-of select="$top-level"/>
    </xsl:message>

    <!-- If there's an anchor, find it -->
    <xsl:variable name="a" select="$contentseq/h:a[1]"/>

    <xsl:choose>
      <xsl:when test="$a">
        <xsl:variable name="pos" select="index-of($contentseq, $a)[1]"/>
        <xsl:message use-when="false()">pos: <xsl:value-of select="$pos"/></xsl:message>
        <xsl:if test="$pos &gt; 1">
          <xsl:copy-of select="subsequence($contentseq, 1, $pos - 1)"/>
        </xsl:if>

        <a>
          <xsl:for-each select="$a/@*">
            <xsl:choose>
              <xsl:when
                test="
                  $top-level = 0
                  and ((namespace-uri(.) = ''
                  and local-name(.) = 'id')
                  or name(.) = 'xml:id')">
                <!-- don't output that attribute -->
              </xsl:when>
              <xsl:otherwise>
                <xsl:copy-of select="."/>
              </xsl:otherwise>
            </xsl:choose>
          </xsl:for-each>

          <xsl:message use-when="false()">
            <xsl:text>children: </xsl:text>
            <xsl:value-of select="count($a/node())"/>
          </xsl:message>

          <xsl:copy-of select="f:unwrap-anchors($a/node(), 1)"/>
        </a>
        <xsl:copy-of select="f:unwrap-anchors(subsequence($contentseq, $pos + 1), 0)"/>
      </xsl:when>
      <xsl:otherwise>
        <xsl:message use-when="false()">no nested a <xsl:value-of select="count($contentseq)"
          /></xsl:message>
        <xsl:copy-of select="$contentseq"/>
      </xsl:otherwise>
    </xsl:choose>
  </xsl:function>

</xsl:stylesheet>
