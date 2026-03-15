<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:t="http://schvalid016.uri/"
>

<!-- test use of a path expression containing an axis step referring to document-node(element(X)) -->
<?spec xpath#id-matching-item?>

<xsl:import-schema namespace="http://schvalid016.uri/" schema-location="validation-14.xsd"/>

<xsl:variable name="u">
 <xsl:copy-of select="/" validation="preserve"/>
 **
</xsl:variable>

<xsl:template match="/">
<z>
  <xsl:variable name="c" select="//t:content"/>
  <f match="yes"><xsl:apply-templates select="$c/ancestor::document-node(schema-element(t:test))" mode="m"/></f>
  <g match="no"><xsl:apply-templates select="$c/ancestor::document-node(element(t:content, t:testType))" mode="m"/></g>  
  <h match="no"><xsl:apply-templates select="$u//content/ancestor::document-node(element(t:content, t:testType))" mode="m"/></h>    
</z>
</xsl:template>

<xsl:template match="/" mode="m">
  <a>found</a>
</xsl:template>



</xsl:stylesheet>
