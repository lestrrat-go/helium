<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:fun="http://www.funJobs.com" xmlns:easy="http://www.easyJobs.com"
      xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
      <!-- Purpose: Test case that uses inherit-namespaces = "yes" inherit-namespaces = "no"
  				on different xsl:element instructions to verify the effect of the 
  				inherit-namespaces attribute on differents levels of elements 
  				(parent, child, grandchild). -->

      <t:template match="doc">
            <out>
                  <t:element xmlns="http://www.dummy_ns.com" name="profession"
                        inherit-namespaces="no">
                        <t:element name="IT_Architect"/>
                        <t:element name="IT_Specialist" namespace="http://www.all.com"
                              inherit-namespaces="yes">
                              <dogsbody/>
                        </t:element>
                        <t:element name="Business_Analyst" namespace=""/>
                  </t:element>
            </out>
      </t:template>
</t:transform>
