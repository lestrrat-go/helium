<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:hat="http://hats.uri/"
        exclude-result-prefixes="xs hat">

<!-- PURPOSE: Test "cast as" and "instance of" with user-defined atomic types. -->

  
<xsl:import-schema namespace="http://hats.uri/" schema-location="schema030.xsd"/>

<xsl:param name="p" select="8"/>

  <xsl:template match="/">
    <out>
      <hatsize>
      <t><xsl:value-of select="('8' cast as hat:bighatsize) instance of hat:bighatsize+"/></t> 
      <t><xsl:value-of select="('8' cast as hat:bighatsize?) instance of hat:hatsize*"/></t> 
      <t><xsl:value-of select="('8' cast as hat:bighatsize?) instance of xs:integer?"/></t>       
      <t><xsl:value-of select="(() cast as hat:bighatsize?) instance of xs:string*"/></t> 
      <f><xsl:value-of select="(8 cast as hat:bighatsize) instance of xs:int"/></f> 
      <f><xsl:value-of select="($p cast as hat:bighatsize) instance of element()"/></f> 
      </hatsize>               
    </out>
  </xsl:template>
  
</xsl:stylesheet>