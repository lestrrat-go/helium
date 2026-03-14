<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- PURPOSE:  "." used when not a node - static error even though never executed -->
<?spec xpath#errors?><?error XPTY0004?>

<xsl:template match="/">
<a/>
  <xsl:value-of select="f:doc()" xmlns:f="http://f.com/"/>
<xsl:message>***** error not detected! *****</xsl:message>
</xsl:template>

<xsl:function name="f:doc" xmlns:f="http://f.com/">
  <xsl:for-each select="1 to 5">
    <xsl:value-of select="(1 to 5)[name() = 'a']"/>
  </xsl:for-each>
</xsl:function>
</xsl:stylesheet>