<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/XSL/Transform"
             xmlns:a="http://example.org/test"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xpath-default-namespace="http://www.w3.org/1999/XSL/Transform"
             version="2.0">
<!-- Purpose: Test to show that empty string is returned if the argument in system-property() 
  				does not have a prefix, (even if the default namespace is set to the 'xsl' namespace 
  				and the keyword is correct).-->

   <t:template match="/">
      <a:out>
         <t:value-of select="system-property('version')"/>
      </a:out>
   </t:template>
</t:transform>
