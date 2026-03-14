<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                >

<!-- BUG: Saxon 5.3.2/003 -->
<!-- DESCRIPTION: If an element node is output while instantiating the content of xsl:comment
    or xsl:processing-instruction, the element node and its content should be ignored. The
    element node is ignored but its content is output. -->

<xsl:template match="/">
<a><xsl:comment>It's (<b name="fred" desc="bill">super</b>)</xsl:comment></a>
</xsl:template>
</xsl:stylesheet>
