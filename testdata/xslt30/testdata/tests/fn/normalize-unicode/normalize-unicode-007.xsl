<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- test normalize-unicode() -->
  <!-- test all normalization forms on a string containing a mixture of different forms -->

  <xsl:template match="/">
    <out>
      <xsl:variable name="in" select="'Affinity A&#xFB03;nity'"/>
      <nfc><xsl:value-of select="string-to-codepoints(normalize-unicode($in, 'NFC'))"/></nfc>
      <nfd><xsl:value-of select="string-to-codepoints(normalize-unicode($in, 'NFD'))"/></nfd>
      <nfkc><xsl:value-of select="string-to-codepoints(normalize-unicode($in, 'NFKC'))"/></nfkc>
      <nfkd><xsl:value-of select="string-to-codepoints(normalize-unicode($in, 'NFKD'))"/></nfkd>
    </out>
  </xsl:template>
    

</xsl:stylesheet>
