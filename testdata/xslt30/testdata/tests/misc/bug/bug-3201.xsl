<?xml version="1.0"?>

<xsl:stylesheet  version="2.0" 
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform">


<?spec xslt#disable-output-escaping?>
<!-- BUG: Saxon 5.4.1/007 -->
<!-- DESCRIPTION:
    disable-output-escaping does not work correctly when applied to text output before the
    first start tag, unless xsl:output is used to set the output method explicitly. -->

<xsl:variable name="s1">&lt;output-escaping&gt;</xsl:variable>
<xsl:variable name="s2">&lt;/output-escaping&gt;</xsl:variable>

<xsl:template match="/">
   <xsl:value-of select="$s1" disable-output-escaping="yes" />
   <xsl:text>content</xsl:text>
   <xsl:value-of select="$s2" disable-output-escaping="yes" />
</xsl:template>

</xsl:stylesheet>

