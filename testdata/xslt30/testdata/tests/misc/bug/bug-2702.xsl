<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:ora="http://www.oracle.com/XSL/Transform/java/"
   xmlns:saxon="http://icl.com/saxon"
   xmlns:xalan="http://org.apache.xalan.xslt.extensions.Redirect/"
   extension-element-prefixes="saxon xalan ora">

<?spec xslt#designating-extension-namespace?>
<!-- BUG: Saxon 5.4/003 -->
<!-- DESCRIPTION:
    If a top-level element with a non-null namespace URI appears in the stylesheet,
    and its namespace is designated as an extension element namespace, and Saxon does not
    recognize the namespace URI as one that identifies an extension element implementation,
    then Saxon will reject the element with an error.
    It should ignore it, because a top-level element can never be
    an extension element. -->

<!-- Test top-level elements are ignored -->

<ora:output name="oout" method="html"/>

<xsl:template match="/">
<out>OK</out>
</xsl:template>

</xsl:stylesheet>



