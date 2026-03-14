<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:my="my" 
  exclude-result-prefixes="#all">
  
  <xsl:variable name="data">
    <a>
     <b level="1" type="A"/>
     <b level="1" type="B"/>
    </a>
  </xsl:variable>
  
  <xsl:template name="main">
    <xsl:apply-templates select="$data"/>
  </xsl:template>  

  <xsl:template match="a">
    <xsl:copy>
      <xsl:copy-of select="@*" />
      <!-- If you supply node() instead of * (and change element() to node() in my:f),
           the resulting a element will have only text content in 9.5 if the input
           contains text nodes besides b elements. --> 
      <xsl:sequence select="my:f(*, 1)" />
    </xsl:copy>
  </xsl:template>

  <xsl:function name="my:f" as="element()*">
    <xsl:param name="nodes" as="element()*" />
    <!-- level doesn't have to be xs:integer; bug is reproducible with xs:string 
         or no type declaration at all: -->
    <xsl:param name="level" as="xs:integer" />
    <xsl:choose>
      <xsl:when test="$nodes/self::b">
        <!-- uncomment this in order to make the bug disappear in 9.5: -->
        <!--<xsl:message select="$level"/>-->
        <xsl:for-each-group select="$nodes" group-starting-with="b[xs:integer(@level) = $level]">
          <xsl:copy>
            <xsl:attribute name="current-group-count" select="count(current-group())"/>
            <xsl:attribute name="current-group-names" select="current-group()/name()"/>
            <xsl:copy-of select="@*"/>
            <xsl:sequence select="my:f(current-group()[position() gt 1], 1)"/>
          </xsl:copy>
        </xsl:for-each-group>
      </xsl:when>
      <xsl:otherwise>
        <xsl:sequence select="$nodes" />
      </xsl:otherwise>
    </xsl:choose>
  </xsl:function>

</xsl:stylesheet>