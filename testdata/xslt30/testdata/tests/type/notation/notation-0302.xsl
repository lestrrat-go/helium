<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:a="http://notation.example.com"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs a" version="2.0">

   <!-- Purpose: Test 'ne' on two attributes of the same type derived from xs:NOTATION. The default namespace is 
  				used for some attribute values.The schema is not imported.Check that the attributes are of type 
  				xs:NOTATION. The schema has a @targetNamespace.Cases tested:
				-attributes have enumeration values with matching prefix and local name, returns false
				-attributes have enumeration values with different prefix but same namespace binding and same local name, returns false
				-attributes have enumeration values with different prefix and matching local name, returns true
				-attributes have enumeration values with matching prefix but different local name, returns true-->

   <xslt:import-schema namespace="http://notation.example.com"
      schema-location="namespaceNotationTest.xsd"/>

   <xslt:template match="/a:NOTATION-enumeration">
      <out>
         <types>
            <xslt:for-each select="a:NOTATION-element">
               <type name="{@name}">
                  <xslt:value-of select="data(./@NOTATION-attribute) instance of xs:NOTATION"/>
               </type>
            </xslt:for-each>
         </types>
         <case1>
            <xslt:value-of
               select="a:NOTATION-element[@name='a']/@NOTATION-attribute ne a:NOTATION-element[@name='b']/@NOTATION-attribute"
            />
         </case1>
         <case2>
            <xslt:value-of
               select="a:NOTATION-element[@name='a']/@NOTATION-attribute ne a:NOTATION-element[@name='c']/@NOTATION-attribute"/>
            <xslt:value-of
               select="a:NOTATION-element[@name='a']/@NOTATION-attribute ne a:NOTATION-element[@name='f']/@NOTATION-attribute"
            />
         </case2>
         <case3>
            <xslt:value-of
               select="a:NOTATION-element[@name='a']/@NOTATION-attribute ne a:NOTATION-element[@name='d']/@NOTATION-attribute"/>
            <xslt:value-of
               select="a:NOTATION-element[@name='d']/@NOTATION-attribute ne a:NOTATION-element[@name='f']/@NOTATION-attribute"
            />
         </case3>
         <case4>
            <xslt:value-of
               select="a:NOTATION-element[@name='a']/@NOTATION-attribute ne a:NOTATION-element[@name='e']/@NOTATION-attribute"
            />
         </case4>
      </out>
   </xslt:template>

</xslt:transform>
