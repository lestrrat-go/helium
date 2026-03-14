<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

<?spec xslt#patterns?>
    <!-- Purpose: filtered union in pattern -->
  

<xsl:template match="/">
 <out>
 	<xsl:apply-templates/>
 </out>
</xsl:template>

<xsl:template match="(foo|baz)[*]">
 <parental-foo-baz>
    <xsl:apply-templates/>
 </parental-foo-baz>
</xsl:template>

<xsl:template match="foo|baz">
    <childless-foo-baz>
        <xsl:apply-templates/>
    </childless-foo-baz>
</xsl:template>



</xsl:stylesheet>
