<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>

<xsl:output method="xml" encoding="iso-8859-1"/>

<?spec xslt#constructing-simple-content?>
<!-- BUG: Saxon 5.3.2/001 -->
<!-- DESCRIPTION:
     test recovery action when outputting element nodes into attributes, comments, and PIs -->
     <?same-as-1.0 no?>


<xsl:template match="/">
    <doc spec-changed="2004-12-01">
    <xsl:attribute name="test">attribute <v>!!</v> value</xsl:attribute>
    +<xsl:comment>Comment <v>!!</v> text</xsl:comment>+
    +<xsl:processing-instruction name="pi">pi <v>!!</v> text</xsl:processing-instruction>+
    </doc>
</xsl:template>

</xsl:stylesheet>

