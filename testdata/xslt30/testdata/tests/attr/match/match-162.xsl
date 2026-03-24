<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test of template matching with the union document-node(element($name,$type)) | element($name,$type) 
  				that will both match the document element. Show that the template is executed twice.-->

   <t:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <t:template
      match="document-node(element(my:userNode, my:userType))|element(my:userNode, my:userType)">
      <out>
         <E>
            <t:value-of select=". instance of document-node()"/>
            <t:text> * </t:text>
            <t:value-of select="./*[1]/name()"/>
         </E>
         <t:apply-templates select="./*"/>
      </out>
   </t:template>

   <t:template match="element(*)"> </t:template>
</t:transform>
