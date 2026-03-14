<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
default-validation="preserve">

  <!-- PURPOSE: Test validation="preserve" on xsl:copy-of. -->
    <?spec xslt#copy-of?>
    <?spec xslt#validating-using-validation-attribute?> 
    <?spec xslt#annotation-for-constructed-element?> 
    
<xsl:variable name="t0">
  <e1 xsl:type="xs:ID">A001</e1>
  <e2 xsl:type="xs:string">A002</e2>
  <e3 xsl:type="xs:integer">003</e3>
  <e4 xsl:type="xs:date">2002-10-20</e4>
</xsl:variable>

<xsl:variable name="tree">
  <xsl:copy-of select="$t0" validation="preserve"/>
</xsl:variable>            

  <xsl:template match="/">
    <out>
      <text test="{data($tree/e3) instance of xs:integer}" value="{data($tree/e3)}"/>
    </out>
  </xsl:template>

</xsl:stylesheet>