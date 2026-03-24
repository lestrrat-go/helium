<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
<!-- Purpose: Tests undeclare-prefixes="false" on xsl:result-document element where the method is 'xml' and version is '1.1', the namespace is an IRI.  -->

   <t:template match="/">
   	<t:result-document method="xml" output-version="1.1" undeclare-prefixes="false">
		    <out>
         <t:element name="a" inherit-namespaces="yes">
	    		     <t:namespace name="pref">http://example.org/preÀ</t:namespace>
	    		     <t:element name="b"/>
	  		    </t:element>
      </out>
   	</t:result-document>
	  </t:template>
</t:transform>
