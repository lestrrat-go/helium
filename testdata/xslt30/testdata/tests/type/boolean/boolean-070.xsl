<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

<?spec xpath#id-general-comparisons?>
<!-- PURPOSE: Test of "!=" operator with an unordered type (anyURI) -->
<?same-as-1.0 no?>


<xsl:template match="/">
  <xsl:param name="p1" select="(xs:anyURI('one.uri'), xs:anyURI('eins.uri'))"/>
  <xsl:param name="p2" select="(xs:anyURI('two.uri'), xs:anyURI('zwei.uri'))"/>
<out>
  <xsl:value-of select="$p1 != $p2"/> should be true
</out>
</xsl:template>
</xsl:stylesheet>
