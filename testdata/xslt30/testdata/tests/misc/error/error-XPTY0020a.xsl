<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- PURPOSE:  axis step when the context item is not a node - type error detected statically or dynamically -->
<?spec xpath#errors?><?error XPTY0020?>

<xsl:template match="/">
<a b="{f:doc()}" xmlns:f="http://f.com/"/>
</xsl:template>

<xsl:function name="f:doc" xmlns:f="http://f.com/">
  <xsl:for-each select="1 to 5">
    <xsl:sequence select="//a"/>
  </xsl:for-each>
</xsl:function>
</xsl:stylesheet>