<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/ns/various"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test type of global xsl:variable with @select selecting a typed attribute node from input file, 
  				@as="schema-attribute($QName)". Verify that the type annotation for the selected attribute is the 
  				same as the schema attribute declaration $QName and its name= $QName.-->

   <t:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <t:variable name="var1" select="/my:userNode/@my:specialPart"
      as="schema-attribute(my:specialPart)"/>

   <t:template match="/my:userNode">
      <out>
         <var1>
            <t:value-of select="$var1 instance of schema-attribute(my:specialPart)"/>
            <t:value-of select="$var1 instance of attribute(my:specialPart, my:partNumberType)"/>
            <t:value-of select="@my:specialPart instance of schema-attribute(my:specialPart)"/>
            <t:value-of select="data(@my:specialPart) instance of my:partNumberType"/>
         </var1>
      </out>
   </t:template>
</t:transform>
