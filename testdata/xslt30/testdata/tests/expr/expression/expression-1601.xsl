<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<!-- test escape-html-uri function -->

<xsl:template name="main">
  <escape>
    <xsl:for-each select="$data/a">
      <xsl:element name="e{position()}"><xsl:value-of select="escape-html-uri(.)"/></xsl:element>
    </xsl:for-each>
  </escape>                             
</xsl:template>
  
  <xsl:variable name="data">
      <a>simple.url</a>
      <a>uri with spaces.uri</a>
      <a>pre-escaped%20uri.uri</a>
      <a>Ælfred.uri</a>
      <a>Euro&#x20AC;.uri</a>
      <a>Surrogate Pair &#100500;.uri</a>
      <a>http://example.com/query.uri?x=y#fragment</a>   
  </xsl:variable>
 
</xsl:stylesheet>
