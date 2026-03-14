<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://my.net/"
  exclude-result-prefixes="my xs">

  <!-- PURPOSE test initial-template invocations -->
  
  <xsl:param name="a" select="12"/>
  <xsl:variable name="b" select="$a * 2"/>
  
  <xsl:variable name="data" as="element()*">
    <a id="1">123.5</a>
    <a id="2">-18</a>
    <a id="3">12.6</a>
    <a id="4">14.2</a>
    <a id="5">123.60</a>
    <a id="6">-56.0</a>
    <a id="7">123.5</a>
    <a id="8">123.6</a>
  </xsl:variable>

  <xsl:template name="main" as="element()*">
    <out>
      <xsl:copy-of select="$data[@id mod 2 = 0]"/>
    </out>
  </xsl:template>

  <xsl:template name="param1" as="element()*">
    <xsl:param name="a" required="yes"/>
    <xsl:param name="my:b" tunnel="yes"/>
    <a>
      <xsl:sequence select="$a,$my:b,$b"/>
    </a>
  </xsl:template>
  


</xsl:stylesheet>
