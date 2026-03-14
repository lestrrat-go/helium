<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                >

<!-- BUG: Saxon 5.3.1/001 -->
<!-- DESCRIPTION: If an element node is output while instantiating the content of xsl:message, the
    closing ">" or "/>" or the element start tag is omitted. -->
    

<xsl:template match="/">
<a><xsl:message>It's (<b name="fred" desc="bill">super</b>)</xsl:message></a>
</xsl:template>
</xsl:stylesheet>
