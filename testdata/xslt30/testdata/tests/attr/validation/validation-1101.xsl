<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:t="http://ns.example.com/val11/"
>

<!-- PURPOSE: show that if an element is validated as numeric, numeric comparison is OK -->
<?spec xpath#id-general-comparisons?>

<xsl:template match="t:test">
  <a><xsl:value-of select=". > 5" /></a>
</xsl:template>

</xsl:stylesheet>
