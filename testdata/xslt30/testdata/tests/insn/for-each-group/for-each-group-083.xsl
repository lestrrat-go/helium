<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math"
    version="2.0">

    <!--<xsl:variable name="notifications" select="doc('notifications.xml')"/>
        <xsl:variable name="notifications-source-path" select="doc('notifications-source-path.xml')"/>-->

    <xsl:variable name="notifications">
        <Notifications>
            <BatchId>1123213333</BatchId>
            <Notification>
                <NotifId>1</NotifId>
                <EmailNotification>
                    <SenderAddress>abc@def.ghi</SenderAddress>
                    <Subject>SBJ2</Subject>
                </EmailNotification>
            </Notification>
            <Notification>
                <NotifId>2</NotifId>
                <EmailNotification>
                    <SenderAddress>jkl.mno@pqr</SenderAddress>
                    <Subject>SBJ2</Subject>
                </EmailNotification>
            </Notification>
        </Notifications>
    </xsl:variable>
    
    <xsl:variable name="notifications-source-path">
        <Notifications>
            <Notification>
                <NotifId>1</NotifId>
                <MsgText>
                    <![CDATA[notif 1]]>
                </MsgText>
            </Notification>
            <Notification>
                <NotifId>2</NotifId>
                <MsgText>
                    <![CDATA[notif 2]]>
                </MsgText>
            </Notification>
        </Notifications>
    </xsl:variable>
    
    <xsl:template match="*">
        <xsl:copy>
            <xsl:apply-templates/>
        </xsl:copy>
    </xsl:template>
    
    <xsl:template match="text()">
        <xsl:value-of select="normalize-space(.)"/>
    </xsl:template>


    <xsl:template name="main">
        <Notifications>
            <xsl:copy-of select="$notifications//BatchId"/>
            <xsl:for-each-group select="($notifications, $notifications-source-path)/*/Notification" group-by="NotifId">
                    <Notification>
                        <NotifId><xsl:value-of select="current-grouping-key()"/></NotifId>
                        <EmailNotification>
                            <xsl:apply-templates select="current-group()/EmailNotification/*, current-group()/MsgText"/>
                        </EmailNotification>
                    </Notification>
            </xsl:for-each-group>
        </Notifications>
    </xsl:template>


</xsl:stylesheet>
