<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- test optimization of a filter expression whose predicate doesn't depend on the node -->
<?spec xpath#id-predicates?>

<xsl:param name="p" select="2"/>

<xsl:template match="/">
  <out>
    <xsl:copy-of select="//a[$p=3] | //b[$p!=3]  "/>                            
  </out>     
</xsl:template>
 
</xsl:stylesheet>
