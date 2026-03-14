<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
     xmlns:a="http://a.uri/" xmlns:b="http://b.uri/">
     
     <?spec xpath#node-tests?>

  <xsl:template match="/">
    <xsl:element name="o">
      <xsl:value-of select="doc/e/@*:a" separator=","/>
    </xsl:element>
  </xsl:template>
 
</xsl:stylesheet>
