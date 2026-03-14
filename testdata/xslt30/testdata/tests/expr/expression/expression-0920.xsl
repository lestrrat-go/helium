<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- some $x in X satisfies C -->
<?spec xpath#id-quantified-expressions?>

<xsl:template match="/">
  <out>
    <true><xsl:value-of select="some $i in /doc/item satisfies $i/@val=4"/></true> 
    <false><xsl:value-of select="some $i in /doc/item satisfies $i/@val=6"/></false> 
  </out>
</xsl:template>
 
</xsl:stylesheet>
