<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:saxon="http://NOT-REALLY.saxon.sf.net/"
default-validation="preserve"
exclude-result-prefixes="saxon xs">

  <!-- PURPOSE: Test validation="preserve" on xsl:result-document. -->
  <?spec xslt#creating-result-trees?>
  <?spec xslt#validating-document-nodes?>  
  
  <!-- this was originally done by capturing the result document as input to another instance of the stylesheet -->
  
<xsl:output name="reprocess" saxon:next-in-chain="schema013.xsl"/>

<xsl:template match="doc">
<xsl:result-document validation="preserve" format="reprocess">
<e>
  <e1><xsl:element name="id" type="xs:ID">A001</xsl:element></e1>
  <e2><xsl:element name="id" type="xs:string">A002</xsl:element></e2>
  <e3><xsl:element name="id" type="xs:integer">003</xsl:element></e3>
  <e4><xsl:element name="id" type="xs:date">2002-10-20</xsl:element></e4>
</e>
</xsl:result-document>
</xsl:template>            

  <xsl:template match="e">
    <out>
      <string><xsl:value-of select="*[id instance of element(*, xs:string)]/id" separator=" "/></string> 
      <integer><xsl:value-of select="*[id instance of element(*, xs:integer)]/id" separator=" "/></integer>         
      <date><xsl:value-of select="*[id instance of element(*, xs:date)]/id" separator=" "/></date>               
    </out>
  </xsl:template>

</xsl:stylesheet>