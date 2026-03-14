<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with xsl:template that has @as="document-node(element(QName, user-defined-list-type)+". 
  				The sequence constructor contains several xsl:document with a valid document elements.-->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:template match="/">
      <out>
         <xslt:call-template name="temp"/>
      </out>
   </xslt:template>

   <xslt:template name="temp" as="document-node(element(item,my:partNumberType))+">
      <xslt:document type="my:partNumberType">
         <item>111-AB</item>
      </xslt:document>
      <xslt:document type="my:partNumberType">
         <xslt:element name="item">999-AC</xslt:element>
      </xslt:document>
   </xslt:template>
</xslt:transform>
