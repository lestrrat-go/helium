<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#copy-of?>
<!-- BUG: Saxon 5.4.1/009 -->
<!-- DESCRIPTION:
    An IndexOutOfBoundException occurs if the nesting depth of any element within a document
    (that is, the number of ancestors) exceeds 100. -->

<xsl:template match="/"><xsl:copy-of select="."/></xsl:template>
 
</xsl:stylesheet>

