<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that an static error is raised if an element 
  				from the XSLT namespace have an attribute whose namespace is null. -->

   <t:variable name="employee" department="'IT'"/>

   <t:template match="doc">
		    <t:element name="year"/>
	  </t:template>
</t:transform>
