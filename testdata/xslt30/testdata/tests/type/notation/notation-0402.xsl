<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:one="http://notation.example.com"
   xmlns:first="http://notation.example.com" xmlns:two="http://different.notation.example.com"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs one first two"
   version="2.0">
   <!-- Purpose: Test 'ne' on two variables of the same type derived from xs:NOTATION. The schema is imported and 
  				variable values are explicitly constructed.The schema has a @targetNamespace. Check that the variables 
  				are both of the type declared in the schema and xs:NOTATION Cases tested:
				-the variables have enumeration values with matching prefix and local name, returns false
				-the variables have enumeration values with different prefix but same namespace binding and same local name, returns false
				-the variables have enumeration values with different prefix and matching local name, returns true
				-the variables have enumeration values with matching prefix but different local name, returns true-->

   <xslt:import-schema namespace="http://notation.example.com"
      schema-location="namespaceNotationTest.xsd"/>

   <xslt:variable name="mp3a" select="one:not1-NOTATION-enumeration-Type('one:mp3')"/>

   <xslt:variable name="mp3b" select="one:not1-NOTATION-enumeration-Type('one:mp3')"/>

   <xslt:variable name="mp3c" select="one:not1-NOTATION-enumeration-Type('first:mp3')"/>

   <xslt:variable name="mp3d" select="one:not1-NOTATION-enumeration-Type('two:mp3')"/>

   <xslt:variable name="wav" select="one:not1-NOTATION-enumeration-Type('one:wav')"/>

   <xslt:template match="/">

      <out>

         <types>
            <xslt:for-each select="($mp3a, $mp3b, $mp3c, $mp3d, $wav)">
               <type val="{.}" p="{position()}"
                  is1="{. instance of one:not1-NOTATION-enumeration-Type}" 
                  is2="{. instance of xs:NOTATION}"/>
            </xslt:for-each>
         </types>

         <case1>
            <xslt:value-of select="$mp3a ne $mp3b"/>
         </case1>

         <case2>
            <xslt:value-of select="$mp3a ne $mp3c"/>
         </case2>

         <case3>
            <xslt:value-of select="$mp3a ne $mp3d"/>
         </case3>

         <case4>
            <xslt:value-of select="$mp3a ne $wav"/>
         </case4>

      </out>
   </xslt:template>
</xslt:transform>
