<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                >

<?spec xslt#serialization?>
<!-- BUG: Saxon 5.3.2/009 -->
<!-- DESCRIPTION: If the output method is not specified explicitly, special characters such as
     "&lt;" that are output before the first element start tag are not escaped. -->

<xsl:template match="/">
This text contains escaped characters &lt; and &gt;
</xsl:template>
</xsl:stylesheet>
