<?xml version="1.0" encoding="UTF-8" standalone="no"?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
 version="2.0" xmlns:xmi="http://www.omg.org/XMI" 
 xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
 xmlns:schema-write="http://schema.omelet.eclipse.org/schema-write.nxsl" 
 xmlns:umlx-to-schema="http://schema.omelet.eclipse.org/umlx-to-schema.nxsl" 
 xmlns:utilities="http://schema.omelet.eclipse.org/utilities.nxsl" 
 exclude-result-prefixes="#all">
 
 <!-- test effect of attempting to exclude a system namespace (xsi) that's needed -->
 
 <xsl:template match="/">

                <xmi:XMI xsi:type="FragmentedSchema" xmi:version="2.0">

                    <debug>
                        <xsl:attribute name="package">
                            <xsl:value-of select="'fred'"/>
                        </xsl:attribute>
                    </debug>
                </xmi:XMI>
                
</xsl:template>

</xsl:stylesheet>
