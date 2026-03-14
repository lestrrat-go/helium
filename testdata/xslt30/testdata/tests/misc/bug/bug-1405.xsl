<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                >
<?spec ser#text?>
<!-- BUG: Saxon 5.3.2/005 -->
<!-- DESCRIPTION: If an element node is output when the output method is text, the markup should
    be suppressed but this is not done correctly. -->
    
<!-- test generating elements and other nodes with method="text" -->
<xsl:output method="text"/>

<xsl:template match="/">
First line of text
<a att="93.7">element content</a>
<xsl:comment>a comment</xsl:comment>
<xsl:processing-instruction name="pi">data</xsl:processing-instruction>
Last line of text
</xsl:template>
</xsl:stylesheet>
