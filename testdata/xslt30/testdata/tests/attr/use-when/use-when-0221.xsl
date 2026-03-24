<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that no error is raised in respect of the element and 
  				its contents on which a use-when=false() attribute exist.  -->

   <t:template match="*">
	     <out>       
         <t:element name="employee">   		
    		      <t:attribute name="foo:salary" use-when="false()">    			
    			        <t:value-of select="$v">Error </t:value-of>
    		      </t:attribute>
    	    </t:element>        
         <employee name="Error" t:use-when="false()">e2</employee>
      </out>
  </t:template>
</t:transform>
