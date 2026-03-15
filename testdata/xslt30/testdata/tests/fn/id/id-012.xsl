<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                >

<?spec fo#func-id?>
<!-- BUG: occurs using Saxon with Xerces only -->
<!-- DESCRIPTION: IDs defined in DTD internal subset are not recognized -->

<xsl:template match="/">
<out><xsl:value-of select="id('Z')/@id"/></out>
</xsl:template>
</xsl:stylesheet>
