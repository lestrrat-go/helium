<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- PURPOSE:  "." used when not a node - static error even though never executed -->
<?spec xpath#errors?><?error XPTY0020?>

<xsl:template match="/">
<a/>
  <xsl:value-of select="f:doc()" xmlns:f="http://f.com/"/>
<xsl:message>***** error not detected! *****</xsl:message>
</xsl:template>

<xsl:function name="f:doc" xmlns:f="http://f.com/">
  <xsl:analyze-string select="'abcd'" regex="a">
    <xsl:matching-substring>
      <xsl:value-of select="//a"/>
    </xsl:matching-substring>
  </xsl:analyze-string>
</xsl:function>
</xsl:stylesheet>