<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">

<!--            Test 'eq' on two attributes of the same type derived from xs:NOTATION. 
  				The schema is not imported and notations are declared with a local name only.
  				The schema has no @targetNamespace. Check that the attributes are of type xs:NOTATION. 
  				Cases tested:
				-attributes have the same enumeration value, returns true
				-attributes have different enumeration and system identifier values, returns false
				-attributes have different enumeration values but the same system identifier, returns false
				-attributes have different enumeration values but the same public identifier, returns false
				-attributes have different enumeration values but the same public and system identifier, returns false-->

   <xslt:import-schema schema-location="noNamespaceNotation.xsd"/>

   <xslt:template match="/doc">
      <out>
         <types>
            <xslt:for-each select="sub">
              <type n="{./@name}" v="{data(./@filetype) instance of xs:NOTATION}"/>
		    </xslt:for-each>
         </types>
         <case1>
            <xslt:value-of select="sub[@name='a']/@filetype eq sub[@name='b']/@filetype"/>
         </case1>
         <case2>
            <xslt:value-of select="sub[@name='a']/@filetype eq sub[@name='c']/@filetype"/>
         </case2>
         <case3>
            <xslt:value-of select="sub[@name='a']/@filetype eq sub[@name='e']/@filetype"/>
         </case3>
         <case4>
            <xslt:value-of select="sub[@name='c']/@filetype eq sub[@name='e']/@filetype"/>
         </case4>
         <case5>
            <xslt:value-of select="sub[@name='a']/@filetype eq sub[@name='d']/@filetype"/>
         </case5>
      </out>
   </xslt:template>
</xslt:transform>
