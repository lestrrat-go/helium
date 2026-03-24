<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        exclude-result-prefixes="xs">

  <!-- PURPOSE: Test use of attribute(*,T) in a pattern. -->
  <?spec xslt#patterns?>
  <?spec xpath#id-attribute-test?>

<xsl:variable name="tree">
  <e1><xsl:attribute name="id" type="xs:ID">A001</xsl:attribute></e1>
  <e2><xsl:attribute name="id" type="xs:string">A002</xsl:attribute></e2>
  <e8><xsl:attribute name="id" type="xs:int">93</xsl:attribute></e8>  
</xsl:variable>            

  <xsl:template match="/">
    <out>
      <xsl:apply-templates select="$tree/*/@id"/>       
    </out>
  </xsl:template>
  
  <xsl:template match="attribute(*, xs:ID)" priority="2">
    <ID parent="{name(..)}"/>
  </xsl:template>
  
  <xsl:template match="attribute(*, xs:string)" priority="1">
    <string parent="{name(..)}"/>
  </xsl:template> 
  
  <xsl:template match="attribute(*, xs:integer)">
    <integer parent="{name(..)}"/>
  </xsl:template>   

</xsl:stylesheet>