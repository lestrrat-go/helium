<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                 version="2.0">

<?spec fo#func-tokenize?>

<xsl:param name="urlPath" select="'/careers/sites/en/page.htm'" />

<xsl:template match="/">
<z>
   <xsl:variable name="tokenizedUrlPath" select="tokenize($urlPath, '/')"/>
   <xsl:variable name="path" 
select="string-join($tokenizedUrlPath[position() &lt; last()], '/')"/>
   <xsl:variable name="fileName" select="$tokenizedUrlPath[last()]"/>
   <a><xsl:value-of select="$path" /></a>
   <a><xsl:value-of select="$fileName" /></a>
</z>
</xsl:template>

</xsl:stylesheet>
