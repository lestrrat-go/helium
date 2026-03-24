<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <!-- test normalize-unicode() -->
  <!-- test Latin Capital D with dot above -->
  
  <xsl:template match="/">
  <xsl:variable name="c" select="'&#x1e0b;'"/>
  <xsl:variable name="d" select="'&#x64;&#x0307;'"/>
  <!-- 1E0B;1E0B;0064 0307;1E0B;0064 0307;-->
    <out>
      <nfc><xsl:value-of select="string-to-codepoints(normalize-unicode($c, 'NFC'))"/></nfc>
      <nfc><xsl:value-of select="string-to-codepoints(normalize-unicode($d, 'NFC'))"/></nfc>
      <nfd><xsl:value-of select="string-to-codepoints(normalize-unicode($c, 'NFD'))"/></nfd>
      <nfd><xsl:value-of select="string-to-codepoints(normalize-unicode($d, 'NFD'))"/></nfd>
    </out>
  </xsl:template>
    

</xsl:stylesheet>
