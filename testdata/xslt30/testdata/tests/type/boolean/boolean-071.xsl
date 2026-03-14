<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-value-comparisons?>
<!-- PURPOSE: Test of eq (etc) with an empty sequence as operand -->
<?same-as-1.0 no?>


<xsl:template match="/">
  <xsl:param name="p1" select="/*/set1/value[1]"/>
  <xsl:param name="p2" select="/*/set2/value[1]"/>
  <xsl:param name="p3" select="/*/set3/value[1]"/> <!-- empty -->
<out>
  <xsl:value-of select="count($p1 eq $p2), count($p1 eq $p3), count($p2 ne $p3), count($p3 ge $p3)"/> 
</out>
</xsl:template>
</xsl:stylesheet>
