<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#creating-attributes?>
    <!-- Purpose: Successful validation against built-in declarations for the XML namespace. -->
   
   <xsl:import-schema namespace="http://www.w3.org/XML/1998/namespace"/>

<xsl:template match="/" name="xsl:initial-template">
   <out>
      <xsl:variable name="temp">
         <e1>
            <xsl:attribute name="xml:lang" select="'en-US'" validation="strict"/>
         </e1>
         <e2>
            <xsl:attribute name="xml:lang" select="''" validation="strict"/>
         </e2>
         <e3>
            <xsl:attribute name="xml:space" select="'preserve'" validation="strict"/>
         </e3>
         <e4>
            <xsl:attribute name="xml:space" select="'default'" validation="strict"/>
         </e4>
         <e5>
            <xsl:attribute name="xml:id" select="'ID001'" validation="strict"/>
         </e5>
      </xsl:variable>
      <xsl:copy-of select="$temp"/>
      <id-ok><xsl:value-of select="exists(id('ID001', $temp)) and not(exists(id('ID002', $temp)))"/></id-ok>
   </out>
</xsl:template>

</xsl:stylesheet>