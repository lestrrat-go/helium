<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

<!-- DOCUMENT: http://www.w3.org/TR/xpath -->
<!-- SECTION: 3.4 Booleans -->
<!-- PURPOSE: Test of "=" operator with mixed types -->
<!-- String to number comparison is not allowed except in backwards compatibility mode -->
<?spec xpath#errors?><?error XPTY0004?>

<xsl:template match="/">
  <xsl:param name="set1" select="('1.0', '3.0')" as="xs:string*"/>
  <xsl:variable name="set2" select="(1, '2')"/>
<out>
  <xsl:value-of select="$set1 = $set2"/> should be true
  <xsl:value-of select="$set2 != $set1"/> should be true
  <xsl:value-of select="$set2 &gt; $set1"/> should be false  
  <xsl:value-of select="$set2 &lt; $set1"/> should be true    
</out>
</xsl:template>
</xsl:stylesheet>
