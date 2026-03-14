<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:output method="html" omit-xml-declaration="yes" indent="yes"/>
  
  <!-- xsl:key with contained template rule -->
  
  <xsl:key name='n' match="p" use=".//term">
    <xsl:template match="/">
      <xsl:value-of select="text()"/>
    </xsl:template>
  </xsl:key>
  
  <xsl:template match="/">
    <TEST> should fail </TEST>
  </xsl:template>
  
</xsl:stylesheet>