<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                >

<!-- BUG: Saxon 5.3.2/002 -->
<!-- DESCRIPTION: If an element node is output while instantiating the content of xsl:attribute, no
    error is reported, the tag is output incorrectly in escaped form.  -->
    

<xsl:template match="/">
<a><xsl:attribute name="a">It's (<b name="fred" desc="bill">super</b>)</xsl:attribute></a>
</xsl:template>
</xsl:stylesheet>
