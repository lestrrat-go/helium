<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        exclude-result-prefixes="xsi">

<!-- PURPOSE: Test validation against xs:anyType -->
<?spec xslt#setting-annotation-for-lre?>




  <xsl:template match="/">
    <xsl:variable name="t">
      <tree xsl:type="xs:anyType" pot="black" kettle="white">
        <a xsl:type="xs:integer">23</a>
        <a xsl:type="xs:date">2004-10-03</a>
        <a xsl:type="xs:byte">12</a>
        <b xsl:type="xs:byte">12</b>
        <x><y xsi:type="xs:integer">9999</y></x> 
      </tree>     
    </xsl:variable>
     
     <out>
     <xsl:copy-of select="$t"/>
     </out>
  </xsl:template>
  
</xsl:stylesheet>