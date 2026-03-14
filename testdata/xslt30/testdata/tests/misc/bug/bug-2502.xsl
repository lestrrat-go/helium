<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>

<?spec xslt#document?>
<!-- BUG: Saxon 5.4/002 -->
<!-- DESCRIPTION:
    A call to "document('foo.xml', /)" fails with the message
    "No base URI available for resolving relative URI". The same error occurs
    with any other second argument that refers to the root node of the principal
    source document. -->

<xsl:template match="list">
<out>
    <xsl:copy-of select="document('bug42.xml', /)"/>
</out>
</xsl:template>

</xsl:stylesheet>

