<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">
                 <!--It is a static error if an
                     xsl:import-schema element that contains an
                     xs:schema element has a schema-location attribute,
                     or if it has a namespace attribute that conflicts with the target
                     namespace of the contained schema.-->
   
   <xsl:import-schema namespace="http://example.com/foo">
     <xs:schema targetNamespace="http://example.com/bar">
       <xs:element name="foo"/>
     </xs:schema>
   </xsl:import-schema>
   
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
</xsl:stylesheet>
