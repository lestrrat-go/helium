<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- every $x in X satisfies C -->
<?spec xpath#id-quantified-expressions?>

<xsl:template match="/">
  <out>
    <true><xsl:value-of select="every $i in /doc/item satisfies $i/@val &lt; 6"/></true> 
    <false><xsl:value-of select="every $i in /doc/item satisfies $i/@val = 4"/></false> 
  </out>
</xsl:template>
 
</xsl:stylesheet>
