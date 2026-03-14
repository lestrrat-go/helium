<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math"
    version="3.0" expand-text="yes">

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
    
    <xsl:mode on-no-match="shallow-copy"/>
    
    <xsl:template match="text()">
        <xsl:value-of select="normalize-space(.)"/>
    </xsl:template>


    <xsl:template name="xsl:initial-template">
        <Notifications>
            <xsl:copy-of select="$notifications//BatchId"/>
            <xsl:merge>
                <xsl:merge-source name="notifications" select="$notifications/*/Notification">
                    <xsl:merge-key select="NotifId"/>
                </xsl:merge-source>
                <xsl:merge-source name="notifications-source-path"
                    select="$notifications-source-path/*/Notification">
                    <xsl:merge-key select="NotifId"/>
                </xsl:merge-source>
                <xsl:merge-action>
                    <Notification>
                        <NotifId>{(current-merge-group()/NotifId)[1]}</NotifId>
                        <EmailNotification>
                            <xsl:apply-templates select="current-merge-group()/EmailNotification/*, current-merge-group()/MsgText"/>
                        </EmailNotification>
                    </Notification>
                </xsl:merge-action>
            </xsl:merge>
        </Notifications>
    </xsl:template>


</xsl:stylesheet>
