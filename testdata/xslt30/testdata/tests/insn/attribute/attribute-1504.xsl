<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#creating-attributes?>
    <!-- Purpose: Unsuccessful validation against built-in declarations for the XML namespace. -->

   <xsl:import-schema namespace="http://www.w3.org/XML/1998/namespace"/>
   
   <xsl:template match="/" name="xsl:initial-template">
      <out>
         <xsl:variable name="temp">
            <e5>
               <xsl:attribute name="xml:id" select="'001-ID'" validation="strict"/>
            </e5>
         </xsl:variable>
         <xsl:copy-of select="$temp"/>
      </out>
   </xsl:template>

</xsl:stylesheet>