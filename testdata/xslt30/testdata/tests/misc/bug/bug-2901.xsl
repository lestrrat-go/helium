<xsl:stylesheet version='2.0' xmlns:xsl='http://www.w3.org/1999/XSL/Transform'>
<!-- Written by Santiago Pericas-Geertsen -->

<?spec xpath#id-path-expressions?>
<!-- BUG: Saxon 5.4/007 -->
<!-- DESCRIPTION: When the parent axis is used on the right-hand side of a "/" operator,
    the node-set returned will never include the root node. For example,
    if the current node is the document element, the expression "./.." will return an empty
    node-set. (The problem does not occur if this is simplified to "..") -->



<xsl:template match="A">
<out>
    <a><xsl:value-of select="count(..)"/>;</a>
    <b><xsl:value-of select="count(./..)"/>;</b>
</out>    
</xsl:template>

</xsl:stylesheet>

