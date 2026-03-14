<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
default-validation="preserve">

  <!-- PURPOSE: Test declaring a variable with type "element/attribute(*, X)". -->
  <?spec xpath#id-element-test?>
  <?spec xpath#id-attribute-test?>
  <?spec xslt#variables?>  

<xsl:variable name="tree">
  <e1><xsl:attribute name="id" type="xs:ID">A001</xsl:attribute></e1>
  <e2 xsl:type="xs:NMTOKEN">A002</e2>
</xsl:variable>

<xsl:variable name="v1" select="$tree/e1/@id" as="attribute(*, xs:string)"/>
<xsl:variable name="v2" select="$tree/e2" as="element(*, xs:string)"/>            

  <xsl:template match="/">
    <out val="{($v1, $v2)}"/>
  </xsl:template>

</xsl:stylesheet>