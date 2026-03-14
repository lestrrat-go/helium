<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
                
<!--            Test 'ne' on two attributes of the same type derived from xs:NOTATION. The schema is not 
  				imported and notations are declared with a local name only. The schema has no @targetNamespace.
  				Check that the attributes are of type xs:NOTATION. Cases tested:
				-attributes have the same enumeration value, returns false
				-attributes have different enumeration and system identifier values, returns true
				-attributes have different enumeration values but the same system identifier, returns true
				-attributes have different enumeration values but the same public identifier, returns true
				-attributes have different enumeration values but the same public and system identifier, returns true-->

   <xslt:import-schema schema-location="noNamespaceNotation.xsd"/>

   <xslt:template match="/doc">
      <out>
         <types>
            <xslt:for-each select="sub">
              <xslt:attribute name="{./@name}" select="data(./@filetype) instance of xs:NOTATION"/>
		    </xslt:for-each>
         </types>
         <case1>
            <xslt:value-of select="sub[@name='a']/@filetype ne sub[@name='b']/@filetype"/>
         </case1>
         <case2>
            <xslt:value-of select="sub[@name='a']/@filetype ne sub[@name='c']/@filetype"/>
         </case2>
         <case3>
            <xslt:value-of select="sub[@name='a']/@filetype ne sub[@name='e']/@filetype"/>
         </case3>
         <case4>
            <xslt:value-of select="sub[@name='c']/@filetype ne sub[@name='e']/@filetype"/>
         </case4>
         <case5>
            <xslt:value-of select="sub[@name='a']/@filetype ne sub[@name='d']/@filetype"/>
         </case5>
      </out>
   </xslt:template>

</xslt:transform>
