<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:output indent="yes" encoding="iso-8859-1"/>

  <!-- test normalize-unicode() -->
  <!-- explicit NFC normalization form, trivial examples -->
  <?spec fo#func-normalize-unicode?>
  <xsl:template match="/">
    <out>
      <a><xsl:value-of select="normalize-unicode('The quick brown fox jumped over the lazy dog', 'NFC')"/></a>
      <a><xsl:value-of select="string-to-codepoints(normalize-unicode('The quick brown fox jumped over the lazy dog', 'NFC'))"/></a>        
      <a att="{normalize-unicode('&#x9;&#xa;&#xd;', 'NFC')}"/>
      <a><xsl:value-of select="string-to-codepoints(normalize-unicode('&#x9;&#xa;&#xd;', 'NFC'))"/></a>  
      <a><xsl:value-of select="normalize-unicode('Straßenüberführung', 'NFC')"/></a>  
      <a><xsl:value-of select="string-to-codepoints(normalize-unicode('Straßenüberführung', 'NFC'))"/></a>  
    </out>
  </xsl:template>
    

</xsl:stylesheet>
