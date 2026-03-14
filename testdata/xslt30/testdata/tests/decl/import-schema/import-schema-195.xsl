<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" default-validation="strip">

  <!-- PURPOSE: Test validation on xsl:result-document using xsl:default-validation on an LRE. -->
  <?spec xslt#creating-result-trees?>
  <?spec xslt#validating-document-nodes?>

  <xsl:template match="/">
    <xsl:result-document validation="preserve">
      <e xsl:default-validation="preserve">
        <e1 xsl:type="xs:ID">A001</e1>
        <e2 xsl:type="xs:string">A002</e2>
        <e3 xsl:type="xs:integer">003</e3>
        <e4 xsl:type="xs:date">2002-10-20</e4>
      </e>
    </xsl:result-document>
  </xsl:template>



</xsl:stylesheet>
