<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">


  <!-- test a key with a content constructor rather than a use expression -->
  
  <xsl:strip-space elements="*"/>

  <xsl:key name="k" match="b">
    <xsl:apply-templates select="." mode="kval"/>
  </xsl:key>

  <xsl:template match="b" mode="kval" as="xs:integer">
    <xsl:sequence select="string-length(.)"/>
  </xsl:template>

  <xsl:template match="/">
    <out>
      <four>
        <xsl:copy-of select="key('k', 4)"/>
      </four>
      <three>
        <xsl:copy-of select="key('k', 3)"/>
      </three>
    </out>
  </xsl:template>

</xsl:stylesheet>
