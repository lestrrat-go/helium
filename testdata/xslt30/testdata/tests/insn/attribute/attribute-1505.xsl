<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#creating-attributes?>
    <!-- Purpose: Successful validation against built-in declarations for the XML namespace. 
         (ID constraints not applied with validation='preserve') -->
   
   <xsl:import-schema namespace="http://www.w3.org/XML/1998/namespace"/>

   <xsl:template match="/" name="xsl:initial-template">
      <out>
         <xsl:variable name="temp" as="document-node()">
            <xsl:document validation="preserve">
               <doc>
                  <e5>
                     <xsl:attribute name="xml:id" select="'ID-001'" validation="strict"/>
                  </e5>
                  <e6>
                     <xsl:attribute name="xml:id" select="'ID-001'" validation="strict"/>
                  </e6>
               </doc>
            </xsl:document>
         </xsl:variable>     
         <xsl:copy-of select="$temp"/>
      </out>
   </xsl:template>

</xsl:stylesheet>