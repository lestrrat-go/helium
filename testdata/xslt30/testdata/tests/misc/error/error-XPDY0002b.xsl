<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- PURPOSE:  "." used when undefined - static error even though never executed -->
<?spec xpath#errors?><?error XPDY0002?>

<xsl:template match="/">
<a b="{f:doc()}" xmlns:f="http://f.com/"/>
</xsl:template>

<xsl:function name="f:doc" xmlns:f="http://f.com/">
  <xsl:sequence select="//a"/>
</xsl:function>
</xsl:stylesheet>