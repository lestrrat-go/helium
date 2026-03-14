<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- test non-integer predicates (bug 499030) -->
<?spec xpath#id-predicates?>

<xsl:template match="/">
  <o><xsl:copy-of select="doc/item[1.7]"/></o>
</xsl:template>
 
</xsl:stylesheet>
