<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.schematreatas.example.com/ns/integer"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Show that an invalid use of 'treat as' does not throw an error in part of the stylesheet that is never executed.-->

   <xslt:import-schema namespace="http://www.schematreatas.example.com/ns/integer"
                       schema-location="integerTypesSchemaTreatas.xsd"/>

   <xslt:template match="/my:userNode">
      <out>
         <xslt:if test="false()">
		          <xslt:value-of select="codepoints-to-string(data(elem-decimal) treat as xs:integer)"/>
	        </xslt:if>
      </out>
   </xslt:template>
</xslt:transform>
