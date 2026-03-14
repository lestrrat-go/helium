<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
   xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
   xmlns:xs="http://www.w3.org/2001/XMLSchema">

 <?spec xslt#creating-attributes?>
    <!-- Purpose: Unsuccessful validation against built-in declarations for the XML namespace. 
         (ID constraints are applied with validation='strict') -->

   <xsl:import-schema namespace="http://www.w3.org/XML/1998/namespace"/>
   <xsl:template match="/" name="xsl:initial-template">
      <out>
         <xsl:variable name="temp" as="document-node()">
            <xsl:document validation="strict">
               <doc xsi:type="xs:anyType">
                  <e5>
                     <xsl:attribute name="xml:id" select="'ID-001'" validation="strict"/>
                  </e5>
                  <e6>
                     <xsl:attribute name="xml:id" select="'ID-002'" validation="strict"/>
                  </e6>
                  <e7 xsi:type="xs:IDREFS">ID-001 ID-002 ID-003</e7>
               </doc>
            </xsl:document>
         </xsl:variable>     
         <xsl:copy-of select="$temp"/>
      </out>
   </xsl:template>

</xsl:stylesheet>