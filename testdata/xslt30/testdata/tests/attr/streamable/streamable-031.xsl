<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes=" xs" version="3.0">


  <!-- within a streaming template, use value-of with multiple elements and attributes, and with re-ordering needed -->

  <xsl:mode streamable="yes"/>

  <xsl:output method="xml" indent="yes" encoding="UTF-8"/>

  <xsl:strip-space elements="*"/>

  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="true" href="ot2.xml"><xsl:apply-templates select="."/></xsl:source-document>
    </out>
  </xsl:template>

  <xsl:template match="coverpg|preface|titlepg"/>

  <xsl:template match="*">
    <xsl:copy>
      <xsl:copy-of select="@*"/>
      <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>

  <xsl:template match="book">
    <book>
      <xsl:variable name="temp" as="text()*">
        <xsl:value-of select="chapter/(@nr/string(), @length/string(), chtitle/string()) "/>
      </xsl:variable>
      <nodes>
        <xsl:value-of select="count($temp)"/>
      </nodes>
      <xsl:if test="position() = 1">
        <copy>
          <xsl:for-each select="$temp">
            <t>
              <xsl:value-of select="."/>
            </t>
          </xsl:for-each>
        </copy>
        <node1 length="{string-length($temp[1])}"/>
        <node2 length="{string-length($temp[2])}"/>
        <node3 length="{string-length($temp[3])}"/>
        <node4 length="{string-length($temp[4])}"/>
        <node5 length="{string-length($temp[5])}"/>
        <node6 length="{string-length($temp[6])}"/>
        <node7 length="{string-length($temp[7])}"/>
      </xsl:if>
    </book>
  </xsl:template>

</xsl:transform>
