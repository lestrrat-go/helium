<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- test normalize-unicode() -->
  <!-- explicit NFKC normalization form, trivial examples -->

  <xsl:template match="/">
    <out>
      <a><xsl:value-of select="normalize-unicode('The quick brown fox jumped over the lazy dog', 'NFKC')"/></a>
      <a><xsl:value-of select="string-to-codepoints(normalize-unicode('The quick brown fox jumped over the lazy dog', 'NFKC'))"/></a>        
      <a att="{normalize-unicode('&#x9;&#xa;&#xd;', 'NFKC')}"/>
      <a><xsl:value-of select="string-to-codepoints(normalize-unicode('&#x9;&#xa;&#xd;', 'NFKC'))"/></a>  
      <a><xsl:value-of select="normalize-unicode('Straßenüberführung', 'NFKC')"/></a>  
      <a><xsl:value-of select="string-to-codepoints(normalize-unicode('Straßenüberführung', 'NFKC'))"/></a>  
    </out>
  </xsl:template>
    

</xsl:stylesheet>
