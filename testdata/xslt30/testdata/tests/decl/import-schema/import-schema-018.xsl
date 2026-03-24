<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        exclude-result-prefixes="xs">

  <!-- PURPOSE: Test use of attribute(*,xs:anyAtomicType) in a path expression
       (to match untyped atomic nodes). -->
  <?spec xslt#annotation-for-constructed-attribute?>     
  <?spec xpath#id-attribute-test?>

<xsl:variable name="tree">
  <e1><xsl:attribute name="id" type="xs:ID">A001</xsl:attribute></e1>
  <e2><xsl:attribute name="id" type="xs:string">A002</xsl:attribute></e2>
  <e8><xsl:attribute name="id" type="xs:integer">93</xsl:attribute></e8>  
</xsl:variable>            

  <xsl:template match="/">
    <out>
      <xsl:copy-of select="$tree/*[attribute(*, xs:anyAtomicType)]"/>       
    </out>
  </xsl:template>

</xsl:stylesheet>