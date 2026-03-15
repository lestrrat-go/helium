<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:fun="http://www.funJobs.com"
             xmlns:easy="http://www.easyJobs.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="3.0">
	<!-- Purpose: Test case that uses inherit-namespaces ="01" on an xsl:element instruction (not allowed). -->

   <t:template match="doc">
	     <out>
         <t:element name="profession"
                    namespace="http://www.dummy_ns.com"
                    inherit-namespaces="01">
   			      <t:element name="IT_architect"/>
   			      <t:text>
</t:text>
   			      <t:element name="IT_Specialist"
                       namespace="http://www.all.com"
                       inherit-namespaces="1">
   				        <t:value-of select="namespace::*"/>
   			      </t:element>
   			      <t:text>
</t:text>
   			      <t:element name="Busines_Analyst"/>
   			      <t:text>
</t:text>
   		    </t:element>
         <t:text>
</t:text>
      </out>
	  </t:template>
</t:transform>
